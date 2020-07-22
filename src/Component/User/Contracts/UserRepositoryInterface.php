<?php


namespace Omed\Component\User\Contracts;


use Doctrine\Persistence\ObjectRepository;
use Omed\Component\User\Model\User;

interface UserRepositoryInterface extends ObjectRepository
{
    /**
     * @param array $orders
     * @param int $limit
     *
     * @return User[]
     */
    public function getPagedUserList(array $orders=['username'=>'asc'], int $limit=10);
}