<?php

/*
 * This file is part of the Omed project.
 *
 * (c) Anthonius Munthi <https://itstoni.com>
 *
 * For the full copyright and license information, please view the LICENSE
 * file that was distributed with this source code.
 */

declare(strict_types=1);

namespace Omed\Component\User\Manager;

use Doctrine\ORM\EntityManagerInterface;
use Doctrine\Persistence\ObjectManager;
use Doctrine\Persistence\ObjectRepository;
use Omed\Component\User\Contracts\UserRepositoryInterface;
use Omed\Component\User\Model\UserInterface;
use Omed\Component\User\Util\CanonicalFieldsUpdater;
use Omed\Component\User\Util\PasswordUpdaterInterface;

abstract class UserManager implements UserManagerInterface
{
    /**
     * @var ObjectManager
     */
    private $om;

    /**
     * @var string
     */
    private $class;

    /**
     * @var CanonicalFieldsUpdater
     */
    private $canonicalFieldsUpdater;

    /**
     * @var PasswordUpdaterInterface
     */
    private $passwordUpdater;

    /**
     * UserManager constructor.
     *
     * @param PasswordUpdaterInterface $passwordUpdater
     * @param CanonicalFieldsUpdater   $canonicalFieldsUpdater
     * @param ObjectManager            $om
     * @param string                   $class
     */
    public function __construct(PasswordUpdaterInterface $passwordUpdater, CanonicalFieldsUpdater $canonicalFieldsUpdater, ObjectManager $om, string $class)
    {
        $this->class = $class;
        $this->om = $om;
        $this->canonicalFieldsUpdater = $canonicalFieldsUpdater;
        $this->passwordUpdater = $passwordUpdater;
    }

    /**
     * @return ObjectManager|EntityManagerInterface
     */
    public function getManager()
    {
        return $this->om;
    }


    /**
     * @return UserInterface
     */
    public function createUser()
    {
        return new $this->class();
    }

    /**
     * @param UserInterface $user
     */
    public function updateCanonicalFields(UserInterface $user)
    {
        $this->canonicalFieldsUpdater->updateCanonicalFields($user);
    }

    /**
     * @return ObjectRepository|UserRepositoryInterface
     */
    public function getRepository()
    {
        return $this->om->getRepository($this->class);
    }

    public function storeUser(UserInterface $user, $andFlush = true)
    {
        $om = $this->om;
        $om->persist($user);

        if ($andFlush) {
            $om->flush();
        }
    }

    /**
     * @param UserInterface $user
     */
    public function updatePassword(UserInterface $user)
    {
        $this->passwordUpdater->hashPassword($user);
    }

    /**
     * @param string|int $id
     *
     * @return object|UserInterface|null
     */
    public function findById($id)
    {
        return $this->findUserBy(['id' => $id]);
    }

    /**
     * @param string $username
     *
     * @return object|UserInterface|null
     */
    public function findByUsername(string $username)
    {
        return $this->findUserBy(['username' => $username]);
    }

    /**
     * @param array $criteria
     *
     * @return object|UserInterface|null
     */
    public function findUserBy(array $criteria)
    {
        return $this->getRepository()->findOneBy($criteria);
    }

    /**
     * @return PasswordUpdaterInterface
     */
    public function getPasswordUpdater()
    {
        return $this->passwordUpdater;
    }
}
